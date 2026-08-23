using System.ComponentModel.DataAnnotations.Schema;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Storage.ValueConversion;

namespace Parltube.Crawler;

[Table("crawled_queue", Schema = "crawler")]
public class CrawlQueue
{
    
}



public class CrawlerDbContext : DbContext
{
    public CrawlerDbContext(DbContextOptions options) : base(options) { } 

    protected override void ConfigureConventions(ModelConfigurationBuilder configurationBuilder)
    {
        base.ConfigureConventions(configurationBuilder);

        // Global: All strings have max length 256 by default (override per-property if needed)
        configurationBuilder.Properties<string>()
            .HaveMaxLength(256);

        // Global: All DateTimes stored as timestamptz in PostgreSQL
        configurationBuilder.Properties<DateTime>()
            .HaveConversion<UtcDateTimeConverter>();

        configurationBuilder.Properties<DateTime?>()
            .HaveConversion<NullableUtcDateTimeConverter>();

        // Global: All DateTimeOffsets stored as timestamptz
        configurationBuilder.Properties<DateTimeOffset>()
            .HaveColumnType("timestamptz");

        configurationBuilder.Properties<DateTimeOffset?>()
            .HaveColumnType("timestamptz");

        // Global: All enums stored as strings (uppercase)
        configurationBuilder.Properties<Enum>()
            .HaveConversion<string>();
    }

    protected override void OnConfiguring(DbContextOptionsBuilder optionsBuilder) =>
        optionsBuilder.EnableDetailedErrors().EnableDetailedErrors(); 
}

internal sealed class UtcDateTimeConverter : ValueConverter<DateTime, DateTime>
{
    public UtcDateTimeConverter() : base(
        v => v.Kind == DateTimeKind.Utc ? v : DateTime.SpecifyKind(v, DateTimeKind.Utc),
        v => DateTime.SpecifyKind(v, DateTimeKind.Utc)
    ) {}
}

internal sealed class NullableUtcDateTimeConverter : ValueConverter<DateTime?, DateTime?>
{
    public NullableUtcDateTimeConverter() :
        base(
            v => v.HasValue
                ? (v.Value.Kind == DateTimeKind.Utc ? v : DateTime.SpecifyKind(v.Value, DateTimeKind.Utc))
                : null,
            v => v.HasValue
                ? DateTime.SpecifyKind(v.Value, DateTimeKind.Utc)
                : null)
    {
        
    }
}   